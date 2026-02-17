from flask import Flask, jsonify

app = Flask(__name__)

@app.route("/")
def index():
    return jsonify({"message": "yoyoyo, hello from dummy server"})

@app.route("/health")
def health():
    return jsonify({"code": "200", "status": "ok"})

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000)