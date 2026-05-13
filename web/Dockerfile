FROM node:24-alpine AS web-builder

WORKDIR /app/web

COPY web/package*.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

FROM nginx:1.27-alpine

COPY nginx/bank-service.conf /etc/nginx/conf.d/default.conf
COPY nginx/html/ /usr/share/nginx/html/
COPY --from=web-builder /app/web/dist/ /usr/share/nginx/app/
