import session from "express-session";
import { getRedisClient } from "../redis/client.ts";
import { RedisStore } from "connect-redis";
import { env } from "../config/env.ts";

export async function buildSessionMiddleware() {
	const redisClient = await getRedisClient();
	return session({
		store: new RedisStore({
			client: redisClient,
			prefix: "sess:",
		}),
		name: env.sessionName, // cookie 名，默认 connect.sid
		secret: env.sessionSecret,
		resave: false,
		saveUninitialized: false,
		rolling: false,
		cookie: {
			httpOnly: true,
			sameSite: "lax",
			secure: env.isProd,
			maxAge: env.sessionTtl * 1000,
		},
		// connect-redis 会自动用 cookie 的 maxAge 作为 TTL，也可显式 ttl
		// ttl: env.sessionTtl,
	});
}
