import { Router } from "express";
import { issueCsrfToken } from "../middleware/csrf.ts";

export const csrfRouter = Router();
csrfRouter.get("/csrf/token", issueCsrfToken);
