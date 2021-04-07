FROM nginx:1.18.0-alpine
COPY . /usr/share/nginx/html
COPY site_check.conf /etc/nginx/conf.d/configfile.template

ENV PORT 80
ENV HOST 0.0.0.0
EXPOSE 80
RUN sh -c "envsubst '\$PORT' < /etc/nginx/conf.d/configfile.template > /etc/nginx/conf.d/default.conf"
CMD ["nginx", "-g", "daemon off;"]