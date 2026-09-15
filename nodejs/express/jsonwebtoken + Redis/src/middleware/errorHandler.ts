// import type { RequestHandler } from "express";

// export const errorHandler: RequestHandler = (err, _req, res, _next) => {
// 	if (err.name === "UnauthorizedError") {
// 		res.status(401).json({ error: "invalid_token" });
// 		return;
// 	}
// 	console.error(err);
// 	res.status(500).json({ error: "internal_error" });
// };
