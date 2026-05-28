# ── builder ──────────────────────────────────────────────────────────────────
FROM node:22-alpine AS builder

# Enable pnpm via corepack
RUN corepack enable

WORKDIR /app

# Install dependencies first (layer cache)
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

# Copy source
COPY . .

# NEXT_PUBLIC_* vars are inlined at build time — must be present during build
ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL

RUN pnpm build

# ── runner ────────────────────────────────────────────────────────────────────
FROM node:22-alpine AS runner

ENV NODE_ENV=production
# Next standalone server binds to process.env.HOSTNAME; Docker sets HOSTNAME to the
# container id, which makes it listen only on the container IP. Force 0.0.0.0 so the
# in-container healthcheck (localhost) and all interfaces work.
ENV HOSTNAME=0.0.0.0
ENV PORT=3000

WORKDIR /app

# Copy standalone output
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public

EXPOSE 3000

CMD ["node", "server.js"]
