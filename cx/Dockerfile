FROM node:slim

# Create app directory
RUN mkdir -p /usr/src/app
WORKDIR /usr/src/app

# Install app dependencies
COPY cx/package.json /usr/src/app/
RUN npm install

# Bundle app source
COPY cx/ /usr/src/app
RUN mkdir -p /usr/src/compoxure
COPY compoxure/ /usr/src/compoxure
RUN npm install /usr/src/compoxure

EXPOSE 5000
CMD ["node","proxy.js"]
