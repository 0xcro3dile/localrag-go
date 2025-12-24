"""
PDF Parsing Service for LocalRAG
Clean Architecture: This is a Framework/Driver - outermost layer.
Provides HTTP API for PDF text extraction, called by Go adapter.
"""
import io
import json
import logging
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import parse_qs, urlparse

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(message)s')
logger = logging.getLogger(__name__)

# Try multiple PDF libraries for robustness
try:
    import pypdf
    PDF_LIBRARY = "pypdf"
except ImportError:
    try:
        import pdfplumber
        PDF_LIBRARY = "pdfplumber"
    except ImportError:
        PDF_LIBRARY = None
        logger.warning("No PDF library found. Install: pip install pypdf")


def extract_text_pypdf(pdf_bytes: bytes) -> tuple[str, int]:
    """Extract text using pypdf."""
    reader = pypdf.PdfReader(io.BytesIO(pdf_bytes))
    pages = len(reader.pages)
    text_parts = []
    for page in reader.pages:
        text = page.extract_text()
        if text:
            text_parts.append(text)
    return "\n\n".join(text_parts), pages


def extract_text_pdfplumber(pdf_bytes: bytes) -> tuple[str, int]:
    """Extract text using pdfplumber."""
    import pdfplumber
    text_parts = []
    pages = 0
    with pdfplumber.open(io.BytesIO(pdf_bytes)) as pdf:
        pages = len(pdf.pages)
        for page in pdf.pages:
            text = page.extract_text()
            if text:
                text_parts.append(text)
    return "\n\n".join(text_parts), pages


def extract_text(pdf_bytes: bytes) -> dict:
    """Extract text from PDF bytes."""
    if PDF_LIBRARY is None:
        return {"error": "No PDF library installed", "text": "", "pages": 0}
    
    try:
        if PDF_LIBRARY == "pypdf":
            text, pages = extract_text_pypdf(pdf_bytes)
        else:
            text, pages = extract_text_pdfplumber(pdf_bytes)
        
        return {
            "text": text.strip(),
            "pages": pages,
            "library": PDF_LIBRARY
        }
    except Exception as e:
        return {"error": str(e), "text": "", "pages": 0}


class PDFHandler(BaseHTTPRequestHandler):
    """HTTP handler for PDF parsing requests."""
    
    def log_message(self, format, *args):
        logger.info("%s - %s", self.address_string(), format % args)
    
    def do_GET(self):
        """Health check endpoint."""
        if self.path == "/health":
            self._send_json({"status": "ok", "library": PDF_LIBRARY})
        else:
            self._send_json({"error": "Use POST /parse with PDF data"}, 400)
    
    def do_POST(self):
        """Parse PDF from request body."""
        if self.path != "/parse":
            self._send_json({"error": "Unknown endpoint"}, 404)
            return
        
        content_length = int(self.headers.get('Content-Length', 0))
        if content_length == 0:
            self._send_json({"error": "No PDF data"}, 400)
            return
        
        pdf_bytes = self.rfile.read(content_length)
        result = extract_text(pdf_bytes)
        
        if "error" in result and result["error"]:
            self._send_json(result, 500)
        else:
            logger.info(f"Parsed PDF: {result['pages']} pages, {len(result['text'])} chars")
            self._send_json(result)
    
    def _send_json(self, data: dict, status: int = 200):
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(data).encode())


def main():
    port = 8081
    server = HTTPServer(("localhost", port), PDFHandler)
    logger.info(f"[INFO] PDF Service starting on http://localhost:{port}")
    logger.info(f"   Using library: {PDF_LIBRARY or 'NONE - install pypdf!'}")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        logger.info("Shutting down...")
        server.shutdown()


if __name__ == "__main__":
    main()
