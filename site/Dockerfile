FROM nginx:1.18.0-alpine
COPY . /usr/share/nginx/html
COPY site_check.conf /etc/nginx/conf.d/configfile.template
# .conf file can be visible from this location.
RUN chown -R "$USER":nginx /usr/share/nginx/html
RUN chmod +r -R /usr/share/nginx/html

ENV PORT 80
ENV HOST 0.0.0.0
EXPOSE 80
RUN sh -c "envsubst '\$PORT' < /etc/nginx/conf.d/configfile.template > /etc/nginx/conf.d/default.conf"
CMD ["nginx", "-g", "daemon off;"]