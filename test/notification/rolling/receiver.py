import json
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path


class Receiver(BaseHTTPRequestHandler):
    def do_POST(self):
        if self.path not in ("/old", "/candidate"):
            self.send_error(404)
            return

        payload = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        with receipts.open("a") as output:
            print(json.dumps({"path": self.path, "body": payload}), file=output)

        self.send_response(204)
        self.end_headers()


receipts = Path("/tmp/receipts.ndjson")
receipts.touch()
HTTPServer(("", 8080), Receiver).serve_forever()
