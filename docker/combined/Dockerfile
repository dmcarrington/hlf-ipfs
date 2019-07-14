# DockerfilHLF SDK server
FROM amd64/ubuntu:latest
RUN apt-get update
RUN apt-get install nodejs npm git -y
#RUN npm install -g npm@5.5.1

# install the web app
RUN mkdir -p /server/webApp
WORKDIR /server/webApp

# Copy the package.json and npm install before copying the rest of webApp
# so that we don't have to re-do this for every change.
COPY webApp/package.json /server/webApp/package.json
RUN npm install

# Now copy the rest of webApp
COPY webApp /server/webApp/
ADD config /server/webApp/config
ADD crypto-config /server/webApp/crypto-config

EXPOSE 3000

WORKDIR /server/webApp

CMD ["npm", "start"]