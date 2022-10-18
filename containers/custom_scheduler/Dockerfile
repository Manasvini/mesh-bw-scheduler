# start by pulling the python image
FROM golang

# Copy the required files
COPY ./ /app/

# switch working directory
WORKDIR /app

# install the dependencies and packages in the requirements file
RUN go build -o ./build/main .

EXPOSE 9001
CMD ["/app/build/main"]
