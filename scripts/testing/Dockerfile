FROM node:6.17.1-slim

ARG JS_FILE_PATH

COPY configs.json .
COPY secrets.json .
COPY $JS_FILE_PATH server.js

CMD [ "node", "server.js" ]

EXPOSE 8080
