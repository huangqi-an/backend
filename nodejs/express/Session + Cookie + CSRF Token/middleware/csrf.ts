import type { NextFunction, Request, Response } from "express";
import { generateCsrfToken, safeEqual } from "../utils/crypto.util.ts";

const UNSAFE_METHODS = new Set(["POST", "PUT", "PATCH", "DELETE"]);

export function issueCsrfToken(req: Request, res: Response) {
	if (!req.session.csrfToken) {
		req.session.csrfToken = generateCsrfToken();
	}
	res.json({ csrfToken: req.session.csrfToken });
}

export function csrfGuard(req: Request, res: Response, next: NextFunction) {
	if (!UNSAFE_METHODS.has(req.method)) {
		return next();
	}
	const sessionToken = req.session.csrfToken;
	const clientToken =
		(req.headers["x-csrf-token"] as string | undefined) ||
		(req.body && typeof req.body._csrf === "string"
			? req.body._csrf
			: undefined);

	if (!sessionToken || !clientToken) {
		return res.status(403).json({ error: "CSRF token missing" });
	}
	if (!safeEqual(clientToken, sessionToken)) {
		return res.status(403).json({ error: "CSRF token invalid" });
	}
	next();
}
