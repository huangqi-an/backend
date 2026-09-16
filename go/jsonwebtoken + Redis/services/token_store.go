package services

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrNotFound = errors.New("token record not found")

type RefreshRecord struct {
	UserID   string
	FamilyID string
}

type RotateResult int64

const (
	RotateOk      RotateResult = 0
	RotateInvalid RotateResult = 1 // 记录不存在：过期 / 已登出 / 从未存在
	RotateReused  RotateResult = 2 // 已轮换出去的旧 token 再次出现 → 疑似盗用
)

// 轮换必须原子：检查 used → 检查 active → 写新 → 标旧 → 维护 family，全在一个脚本里。
// 拆成多条命令会有 TOCTOU：两个并发刷新都通过校验，各自发一个新 token。
var rotateLua = redis.NewScript(`
-- KEYS[1]=rt:<oldHash>  KEYS[2]=rt:used:<oldHash>  KEYS[3]=rt:<newHash>  KEYS[4]=rt:family:<fid>
-- ARGV[1]=newHash  ARGV[2]=oldHash  ARGV[3]=uid  ARGV[4]=fid  ARGV[5]=ttlSeconds
if redis.call('EXISTS', KEYS[2]) == 1 then
  return 2
end
if redis.call('EXISTS', KEYS[1]) == 0 then
  return 1
end
redis.call('SET', KEYS[2], ARGV[4], 'EX', ARGV[5])
redis.call('DEL', KEYS[1])
redis.call('SREM', KEYS[4], ARGV[2])
redis.call('HSET', KEYS[3], 'uid', ARGV[3], 'fid', ARGV[4])
redis.call('EXPIRE', KEYS[3], ARGV[5])
redis.call('SADD', KEYS[4], ARGV[1])
redis.call('EXPIRE', KEYS[4], ARGV[5])
return 0
`)

type TokenStore struct {
	rdb        redis.UniversalClient
	prefix     string
	refreshTTL time.Duration
	rotate     *redis.Script
}

func NewTokenStore(rdb redis.UniversalClient, prefix string, refreshTTL time.Duration) *TokenStore {
	return &TokenStore{
		rdb:        rdb,
		prefix:     prefix,
		refreshTTL: refreshTTL,
		rotate:     rotateLua,
	}
}

func (s *TokenStore) activeKey(h string) string {
	return s.prefix + "rt:" + h
}

func (s *TokenStore) usedKey(h string) string {
	return s.prefix + "rt:used:" + h
}
func (s *TokenStore) familyKey(fid string) string {
	return s.prefix + "rt:family:" + fid
}
func (s *TokenStore) blacklistKey(j string) string {
	return s.prefix + "bl:" + j
}

/**集群模式
func (s *TokenStore) activeKey(h, fid string) string {
	return s.prefix + "rt:{" + fid + "}:" + h
}
func (s *TokenStore) usedKey(h, fid string) string {
	return s.prefix + "rt:used:{" + fid + "}:" + h
}
func (s *TokenStore) familyKey(fid string) string {
	return s.prefix + "rt:family:{" + fid + "}"
}
*/

// 登录：写入 active 记录 + 建一条新的 family 链
func (s *TokenStore) SaveRefresh(ctx context.Context, hash, userID, familyID string) error {
	tx := s.rdb.TxPipeline()
	tx.HSet(ctx, s.activeKey(hash), "uid", userID, "fid", familyID)
	tx.Expire(ctx, s.activeKey(hash), s.refreshTTL)
	tx.SAdd(ctx, s.familyKey(familyID), hash)
	tx.Expire(ctx, s.familyKey(familyID), s.refreshTTL)
	_, err := tx.Exec(ctx)
	return err
}

// 注意：这里只读不写，读到的 fid 对该 token 是恒定的（同一随机串永远映射同一条链），
// 所以"先读 fid，再用脚本原子改"不存在正确性问题。
func (s *TokenStore) ActiveRecord(ctx context.Context, hash string) (*RefreshRecord, error) {
	vals, err := s.rdb.HMGet(ctx, s.activeKey(hash), "uid", "fid").Result()
	if err != nil {
		return nil, err
	}
	if len(vals) != 2 || vals[0] == nil || vals[1] == nil {
		return nil, ErrNotFound
	}
	return &RefreshRecord{
		UserID:   vals[0].(string),
		FamilyID: vals[1].(string),
	}, nil
}

// 返回该 token 曾经所属的 family（只对"已被轮换出去"的 token 有值）
func (s *TokenStore) UsedFamily(ctx context.Context, hash string) (string, error) {
	fid, err := s.rdb.Get(ctx, s.usedKey(hash)).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return fid, err
}

// 返回该 token 曾经所属的 family（只对"已被轮换出去"的 token 有值）
func (s *TokenStore) Rotate(ctx context.Context, oldHash, newHash, userID, familyID string) (RotateResult, error) {
	ttl := int64(s.refreshTTL.Seconds())
	res, err := s.rotate.Run(ctx, s.rdb, []string{
		s.activeKey(oldHash),  // KEYS[1] = rt:<oldHash>
		s.usedKey(oldHash),    // KEYS[2] = rt:used:<oldHash>
		s.activeKey(newHash),  // KEYS[3] = rt:<newHash>
		s.familyKey(familyID), // KEYS[4] = rt:family:<fid>
	}, newHash, oldHash, userID, familyID, ttl).Int64()
	if err != nil {
		return RotateInvalid, err
	}
	return RotateResult(res), nil
}

// 当前设备登出
func (s *TokenStore) RevokeOne(ctx context.Context, hash, familyID string) error {
	tx := s.rdb.TxPipeline()
	tx.Del(ctx, s.activeKey(hash))
	tx.SRem(ctx, s.familyKey(familyID), hash)
	_, err := tx.Exec(ctx)
	return err
}

// 吊销整条链：复用检测触发时调用，让该链上所有活跃 refresh 立即作废。
// used 标记故意保留，这样被盗的旧 token 再次出现仍能被识别。
func (s *TokenStore) RevokeFamily(ctx context.Context, familyID string) (int64, error) {
	hashes, err := s.rdb.SMembers(ctx, s.familyKey(familyID)).Result()
	if err != nil {
		return 0, err
	}
	keys := make([]string, 0, len(hashes)+1)
	for _, h := range hashes {
		keys = append(keys, s.activeKey(h))
	}
	keys = append(keys, s.familyKey(familyID))
	return s.rdb.Del(ctx, keys...).Result()
}

// 黑名单只存到 access 自然过期为止，过期后 key 自己消失，不会无限增长
func (s *TokenStore) BlacklistAccess(ctx context.Context, jti string, ttl time.Duration) error {
	if ttl < 0 {
		return nil
	}
	return s.rdb.Set(ctx, s.blacklistKey(jti), "1", ttl).Err()
}

func (s *TokenStore) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	n, err := s.rdb.Exists(ctx, s.blacklistKey(jti)).Result()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}
