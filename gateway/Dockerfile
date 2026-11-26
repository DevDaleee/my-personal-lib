# Imagem do gateway para o ECS: Nginx com a configuração e o app web dentro.
# No Compose isso entra por volume; no ECS não existe volume do nosso computador,
# então tudo precisa estar na imagem.
#
# O build web fica em outro repositório. Ele entra por um "contexto extra"
# chamado app (recurso do BuildKit), sem precisar copiar a pasta para cá:
#   docker build --build-context app=../my-personal-lib-app/build/web -t gateway gateway
FROM nginx:1.27-alpine

COPY nginx.ecs.conf /etc/nginx/conf.d/default.conf
COPY --from=app . /usr/share/nginx/html
