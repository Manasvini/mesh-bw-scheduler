# start by pulling the python image
FROM golang

# Copy the required files
COPY ./go.mod /app/go.mod
COPY ./main.go /app/main.go

# switch working directory
WORKDIR /app

# install the dependencies and packages in the requirements file
RUN go build -o ./build/main .

EXPOSE 6001
CMD ["/app/build/main"]
