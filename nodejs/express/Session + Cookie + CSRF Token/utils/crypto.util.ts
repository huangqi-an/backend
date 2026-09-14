import { randomBytes, timingSafeEqual } from "node:crypto";

export function generateCsrfToken(): string {
	return randomBytes(32).toString("base64url");
}

export function safeEqual(a: string, b: string): boolean {
	const ab = Buffer.from(a);
	const bb = Buffer.from(b);
	if (ab.length !== bb.length) return false;
	return timingSafeEqual(ab, bb); //恒定时间比较，防时序攻击
}
