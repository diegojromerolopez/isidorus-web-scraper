# Stage 1: Build
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json ./
RUN npm install
COPY index.html ./
COPY vite.config.ts ./
COPY tsconfig.json ./
COPY tsconfig.app.json ./
COPY tsconfig.node.json ./
COPY public/ public/
COPY src/ src/
RUN npm run build

# Stage 2: Serve with Node
FROM node:20-alpine
WORKDIR /app
RUN npm install -g serve
COPY --from=builder /app/dist ./dist
CMD ["serve", "-s", "dist", "-l", "8000"]
