# start by pulling the python image
FROM python

# Copy the required files
COPY ./main.py /app/main.py
COPY ./requirements.txt /app/requirements.txt

# switch working directory
WORKDIR /app

# install the dependencies and packages in the requirements file
RUN pip install -r requirements.txt

# configure the container to run in an executed manner
ENTRYPOINT [ "python" ]

EXPOSE 6000
CMD ["main.py"]
