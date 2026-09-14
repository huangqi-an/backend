import { createClient, type RedisClientType } from "redis";
import { env } from "../config/env.ts";

let client: RedisClientType | null = null;

export async function getRedisClient(): Promise<RedisClientType> {
	if (client) return client;
	client = createClient({
		socket: {
			host: env.redis.host,
			port: env.redis.port,
		},
		password: env.redis.password,
	}) as RedisClientType;

	client.on("error", (err) => {
		console.error("[Redis] client error", err);
	});
	await client.connect();
	return client;
}
