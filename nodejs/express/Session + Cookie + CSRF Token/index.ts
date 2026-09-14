import { createApp } from "./app.ts";
import { env } from "./config/env.ts";

async function main() {
	const app = await createApp();
	app.listen(env.port, () => {
		console.log(
			`Server running on http://localhost:${env.port} (${env.nodeEnv})`,
		);
	});
}

main().catch((err) => {
	console.error("Failed to start", err);
	process.exit(1);
});
