FROM python:3.11-slim as builder

# Set working directory
WORKDIR /app


COPY requirements-docs.txt .
RUN pip install --no-cache-dir -r requirements-docs.txt


COPY mkdocs.yml .
COPY docs_site/ ./docs_site/


RUN mkdocs build


FROM nginx:alpine


COPY --from=builder /app/site /usr/share/nginx/html

RUN echo 'server { \
    listen 80; \
    server_name localhost; \
    root /usr/share/nginx/html; \
    index index.html; \
    \
    location / { \
        try_files $uri $uri/ /index.html; \
    } \
    \
    location ~* \.(css|js|png|jpg|jpeg|gif|svg|ico|woff|woff2)$ { \
        expires 1y; \
        add_header Cache-Control "public, immutable"; \
    } \
    \
    add_header X-Frame-Options "SAMEORIGIN" always; \
    add_header X-Content-Type-Options "nosniff" always; \
    add_header X-XSS-Protection "1; mode=block" always; \
}' > /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
