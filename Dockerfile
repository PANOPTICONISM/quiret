FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS backend-builder

WORKDIR /app
COPY backend/go.mod backend/go.sum* ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

FROM alpine:latest

RUN apk --no-cache add poppler-utils

WORKDIR /app

COPY --from=backend-builder /app/server .

COPY --from=frontend-builder /app/frontend/dist ./static

ENV DATA_PATH=/data
ENV STATIC_PATH=/app/static
ENV PORT=8080

EXPOSE 8080

CMD ["./server"]
