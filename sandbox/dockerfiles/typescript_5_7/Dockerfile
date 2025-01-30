FROM node:22

RUN npm install -g typescript@5.7

RUN mkdir -p /app
ADD . /app
WORKDIR /app

ENV PATH /app/node_modules/.bin:$PATH


