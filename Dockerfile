FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY tsconfig.json ./
COPY src/ ./src/
RUN npm run build

FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/dist ./dist
RUN mkdir -p /data /config && chown -R node:node /data /config /app
USER node
VOLUME ["/data"]
CMD ["node", "dist/index.js"]
