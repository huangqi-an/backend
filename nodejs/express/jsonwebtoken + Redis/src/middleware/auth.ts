import type { RequestHandler } from "express";
import { verifyAccessToken } from "../utils/token.ts";
import type { TokenStore } from "../services/tokenStore.ts";

declare global {
	namespace Express {
		interface Request {
			userId?: string;
		}
	}
}

export function authGuard(store: TokenStore): RequestHandler {
	return async (req, res, next) => {
		const header = req.headers.authorization;
		if (!header?.startsWith("Bearer ")) {
			res.status(401).json({ error: "missing_token" });
			return;
		}
		try {
			const payload = verifyAccessToken(header.slice(7));
			if (await store.isBlacklisted(payload.jti)) {
				res.status(401).json({ error: "token_revoked" });
				return;
			}
			req.userId = payload.userId;
			next();
		} catch {
			res.status(401).json({ error: "invalid_or_expired_token" });
		}
	};
}
