# Devis Generator : site statique (HTML/CSS/JS vanilla) servi par nginx
FROM nginx:1.27-alpine

LABEL org.opencontainers.image.title="devisgenerator" \
      org.opencontainers.image.description="Générateur de devis/factures pour micro-entrepreneurs (site statique)" \
      org.opencontainers.image.source="https://github.com/tainalo2/devis_generator" \
      org.opencontainers.image.licenses="GPL-3.0"

COPY docker/nginx.conf /etc/nginx/conf.d/default.conf
COPY index.html main.js style.css /usr/share/nginx/html/
COPY src/ /usr/share/nginx/html/src/

EXPOSE 80
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
  CMD wget -qO- http://127.0.0.1/ >/dev/null || exit 1
