from flask import Flask
import os

app = Flask(__name__)


@app.route('/')
def home():
    if 'MY_NODE_NAME' in os.environ:
        return "Flask App " + os.environ['MY_NODE_NAME'] + "\n"
    else:
        return "Flask App\n"


if __name__ == "__main__":
    port = int(os.environ.get('PORT', 6000))
    app.run(debug=True, host='0.0.0.0', port=port)

