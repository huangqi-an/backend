import { createApp } from "./app.ts";
import { env } from "./config/env.ts";

const app = await createApp();
app.listen(env.port, () =>
	console.log(
		`Server running on http://localhost:${env.port} (${env.nodeEnv})`,
	),
);
