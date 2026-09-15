import { config } from "dotenv";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

// 无论从哪个 CWD 启动，都从项目根目录加载 .env
const __dirname = dirname(fileURLToPath(import.meta.url));
config({
	path: [
		resolve(__dirname, "../../.env"),
		resolve(__dirname, "../../../../../.env"),
	],
});

function getEnv(key: string, defaultValue?: string): string {
	const value = process.env[key];
	if (!value) {
		if (defaultValue !== undefined) return defaultValue;
		throw new Error(`环境变量 ${key} 未定义`);
	}
	return value;
}

export const env = {
	nodeEnv: getEnv("NODE_ENV", "development") as
		| "development"
		| "production"
		| "test",
	port: parseInt(getEnv("PORT", "3000"), 10),
	isProd: getEnv("NODE_ENV", "development") === "production",
	jwtSecret: getEnv("JWT_SECRET"),
	accessExpires: getEnv("ACCESS_TOKEN_EXPIRES"),
	refreshExpiresSec: parseInt(
		getEnv("REFRESH_TOKEN_EXPIRES_SEC", "604800"),
		10,
	),
	redis: {
		host: getEnv("REDIS_HOST", "localhost"),
		port: parseInt(getEnv("REDIS_PORT", "6379"), 10),
		password: getEnv("REDIS_PASSWORD", "") || undefined,
	},
} as const;
