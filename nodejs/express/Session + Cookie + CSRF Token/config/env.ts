import "dotenv/config";

function getEnv(key: string, defaultValue?: string): string {
	const value = process.env[key];
	if (!value) {
		if (defaultValue !== undefined) {
			return defaultValue;
		}
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
	sessionSecret: getEnv("SESSION_SECRET"),
	sessionName: getEnv("SESSION_NAME"),
	redis: {
		host: getEnv("REDIS_HOST", "localhost"),
		port: parseInt(getEnv("REDIS_PORT"), 10),
		password: getEnv("REDIS_PASSWORD") || undefined,
	},
	sessionTtl: parseInt(getEnv("SESSION_TTL", "3600"), 10), // Session TTL in seconds
	isProd: getEnv("NODE_ENV") === "production",
} as const;
