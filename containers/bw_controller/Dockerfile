# start by pulling the python image
FROM scratch
# Copy the required files

COPY ./controller_main ./
COPY ./config_cloudlab_k3s.json ./
# switch working directory

# install the dependencies and packages in the requirements file
ENTRYPOINT [ "./controller_main", "-config", "./config_cloudlab_k3s.json"]
