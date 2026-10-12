FROM golang:1.26-alpine 

WORKDIR /application

COPY . /application

RUN go build -o bin/ratewatch ./cmd/ratewatch

EXPOSE 8083

CMD [ "./bin/ratewatch" ]
