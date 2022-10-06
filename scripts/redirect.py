import os
import sys
import requests
from flask import Flask,request

app = Flask(__name__, static_url_path='/~')

@app.route('/', methods=['GET'], defaults={'path': ''})
@app.route('/<path:path>', methods=['GET'])
def hello(path):
    forwardto = sys.argv[1] + request.full_path
    print(forwardto, file=sys.stderr)
    return requests.get(forwardto).text

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=9091)
