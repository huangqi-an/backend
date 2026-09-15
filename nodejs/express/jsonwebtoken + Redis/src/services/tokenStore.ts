import type { RedisClientType } from "redis";
import { env } from "../config/env.ts";

export type RefreshRecord = { userId: string; familyId: string };

/**
 * K 是 Key builder（Key 生成器），把散落在代码各处的字符串拼接集中管理，避免手写 "rt:" + hash 时拼错导致 bug。
 */
const K = {
	active: (h: string) => `rt:${h}`,
	used: (h: string) => `rt:used:${h}`,
	family: (familyId: string) => `rt:family:${familyId}`,
	blacklist: (jti: string) => `bl:${jti}`,
};

export function createTokenStore(redis: RedisClientType) {
	/**
   * 
   * 从环境变量读取 refresh token 的有效期（秒）。这个值会同时用于：
    单个 refresh token 记录的过期时间
    整个 family 集合的过期时间
   */
	const ttl = env.refreshExpiresSec;

	/** 登录：保存 refresh token + 建立 family 链 */
	async function saveRefresh(hash: string, rec: RefreshRecord) {
		const m = redis.multi(); //创建一个 MULTI 事务。Node Redis 客户端里的 multi() 返回一个事务对象，之后所有命令会排队

		m.set(K.active(hash), JSON.stringify(rec), { EX: ttl }); //把记录序列化成 JSON 存入 rt:<hash>，并设置 EX ttl 秒后自动过期。这样每个 token 都有独立的生存期。

		m.sAdd(K.family(rec.familyId), hash); //把当前 token 的 hash 加入对应的 family Set。同一个 familyId 下会积累整条链上的所有 token hash（包括旧的、已经过期的、已经用过的）。

		m.expire(K.family(rec.familyId), ttl); //给 family Set 刷新过期时间。
		await m.exec();
	}

	/** 轮换：旧 token 标记 used + 删 active；新 token 写 active + 加入 family */
	async function rotate(oldHash: string, newHash: string, rec: RefreshRecord) {
		const m = redis.multi();
		m.set(K.used(oldHash), rec.familyId, { EX: ttl }); // ① 标记旧 token 已用
		m.del(K.active(oldHash)); // ② 删除旧 token 的有效记录
		m.sRem(K.family(rec.familyId), oldHash); // ③ 从 family 集合移除旧 hash
		m.set(K.active(newHash), JSON.stringify(rec), { EX: ttl }); // ④ 写入新 token
		m.sAdd(K.family(rec.familyId), newHash); // ⑤ 新 token 加入 family
		await m.exec();
	}

	async function getActive(hash: string): Promise<RefreshRecord | null> {
		const raw = await redis.get(K.active(hash));
		return raw ? (JSON.parse(raw) as RefreshRecord) : null;
	}

	/** 返回 familyId（若该 token 曾被轮换出，用于复用检测） */
	async function getUsed(hash: string): Promise<string | null> {
		return redis.get(K.used(hash));
	}

	/** 吊销整条 family（复用检测 / 登出全部设备） */
	async function revokeFamily(familyId: string) {
		const hashes = await redis.sMembers(K.family(familyId));
		if (hashes.length) await redis.del(hashes.map((h) => K.active(h)));
		await redis.del(K.family(familyId));
	}

	/** 仅吊销单个 token（当前设备登出） */
	async function revokeOne(hash: string, familyId: string) {
		const m = redis.multi();
		m.del(K.active(hash));
		m.sRem(K.family(familyId), hash);
		await m.exec();
	}

	/** Access Token 黑名单 */
	async function blacklistAccess(jti: string, ttlSec: number) {
		await redis.set(K.blacklist(jti), "1", { EX: ttlSec });
	}
	async function isBlacklisted(jti: string): Promise<Boolean> {
		return (await redis.exists(K.blacklist(jti))) === 1;
	}

	return {
		saveRefresh,
		rotate,
		getActive,
		getUsed,
		revokeFamily,
		revokeOne,
		blacklistAccess,
		isBlacklisted,
	};
}

export type TokenStore = ReturnType<typeof createTokenStore>;
