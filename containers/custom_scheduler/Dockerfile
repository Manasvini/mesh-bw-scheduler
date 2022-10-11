# start by pulling the python image
FROM golang

# Copy the required files
COPY ./go.mod /app/go.mod
COPY ./main.go /app/main.go
COPY ./kubernetes.go /app/kubernetes.go
COPY ./processor.go /app/processor.go
COPY ./types.go /app/types.go

# switch working directory
WORKDIR /app

# install the dependencies and packages in the requirements file
RUN go build -o ./build/main .

EXPOSE 9001
CMD ["/app/build/main"]
