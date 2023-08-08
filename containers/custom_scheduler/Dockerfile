# start by pulling the python image
FROM golang

# Copy the required files
COPY ./custom_scheduler ./
COPY ./cloudlab_config.json ./
# switch working directory

# install the dependencies and packages in the requirements file

EXPOSE 9001
ENTRYPOINT ["./custom_scheduler", "-config", "./cloudlab_config.json"]
