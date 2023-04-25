# start by pulling the python image
FROM scratch

# Copy the required files
ADD ./controller_main ./
COPY ./config_k3s.json ./
# switch working directory

# install the dependencies and packages in the requirements file

CMD ["./controller_main", "-config", "config_k3s.json"]
