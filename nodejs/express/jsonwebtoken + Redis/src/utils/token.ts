import { createHash, randomBytes, randomUUID } from "node:crypto";
import jwt, { type SignOptions } from "jsonwebtoken";
import { env } from "../config/env.ts";

export type AccessTokenPayload = {
	userId: string;
	jti: string; // 唯一 id，用于黑名单
};

/** 生成随机 refresh token：48 字节 -> base64url */
export function generateRefreshToken(): string {
	return randomBytes(48).toString("base64url");
}

/** SHA-256 哈希，作为 Redis key（不明文存储） 
 * 这段代码是在 Node.js 里用 crypto 模块对 `token` 做 SHA-256 哈希：

1. `crypto.createHash('sha256')`：创建一个使用 SHA-256 算法的哈希对象，相当于开了一个“计算器”，准备好接收数据。
2. `.update(token)`：把 `token` 的内容喂进这个计算器。可以多次调用 update，多次喂入的数据会按顺序合并处理，相当于分段输入。
3. `.digest('hex')`：结束计算，输出最终的哈希值。`'hex'` 表示用十六进制字符串格式返回，也就是最常见的 64 位小写字母加数字的组合（SHA-256 固定输出 256 位，转成 hex 就是 64 个字符）。这一步执行后，哈希对象就不能再喂数据了。

整体效果就是：把任意长度的 `token` 变成一串固定长度、不可逆的指纹，常用于验证数据完整性、生成签名或存储口令摘要。这个过程是单向的，没法从结果反推出原始 `token`。

> 补充一点：`token` 通常来自变量（比如用户输入的字符串），哈希前不需要手动转码，直接传进去就行。实际项目中，哈希常用于防篡改，但不等于加密，它不保证数据保密性，只保证数据没被改动。
 * 
*/
export function hashToken(token: string): string {
	return createHash("sha256").update(token).digest("hex");
}

export function newTokenId(): string {
	return randomUUID();
}

/** 签发 Access Token (JWT, 无状态) */
export function signAccessToken(payload: AccessTokenPayload): string {
	return jwt.sign(payload, env.jwtSecret, {
		expiresIn: env.accessExpires as SignOptions["expiresIn"],
	});
}

/** 验证 Access Token，失败抛异常 */
export function verifyAccessToken(
	token: string,
): AccessTokenPayload & { exp: number } {
	return jwt.verify(token, env.jwtSecret) as AccessTokenPayload & {
		exp: number;
	};
}
