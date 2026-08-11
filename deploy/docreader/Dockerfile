# WeKnora docreader + HTTP 桥接镜像
# 构建命令（从 hr-office 项目根目录执行）:
#   docker build -t hr-office-docreader \
#     -f deploy/docreader/Dockerfile \
#     /home/WeKnora/docreader
FROM python:3.10-slim

WORKDIR /app

# 系统依赖（OpenCV/Pillow/pypdfium2 等需要的运行时库）
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgl1-mesa-glx libglib2.0-0 libsm6 libxext6 libxrender-dev \
    libgomp1 wget \
    && rm -rf /var/lib/apt/lists/*

# 复制 WeKnora docreader 源码
COPY . /app/

# 安装 Python 依赖
RUN pip install --no-cache-dir \
    "beautifulsoup4>=4.14.2" \
    "ebooklib>=0.18" \
    "grpcio>=1.78.0" \
    "grpcio-health-checking>=1.78.0" \
    "grpcio-tools>=1.78.0" \
    "lxml>=6.1.0" \
    "markdownify>=0.13.1" \
    "markitdown[docx,pdf,xls,xlsx,pptx]>=0.1.3" \
    "openpyxl>=3.1.0" \
    "pandas>=2.0.0" \
    "xlrd>=2.0.0" \
    "pillow>=12.2.0" \
    "protobuf>=6.33.5" \
    "pydantic>=2.13.4" \
    "pypdf>=6.13.3" \
    "pypdfium2>=5.8.0" \
    "python-docx>=1.2.0" \
    "requests>=2.33.0" \
    "textract==1.5.0" \
    "trafilatura>=2.0.0" \
    && pip cache purge

# 内联 HTTP 桥接脚本（避免外部文件依赖）
# 详见 deploy/docreader/http_bridge.py（源码版本，此处为镜像内联副本）
RUN printf '%s\n' \
    '#!/usr/bin/env python3' \
    '"""WeKnora docreader HTTP 桥接服务。"""' \
    '' \
    'import json, logging, os, re, sys, time' \
    'from http.server import HTTPServer, BaseHTTPRequestHandler' \
    '' \
    'sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))' \
    'from docreader.parser.parser import Parser' \
    '' \
    'logging.basicConfig(level=logging.INFO, format="%(asctime)s [http_bridge] %(levelname)s %(message)s")' \
    'logger = logging.getLogger("http_bridge")' \
    'parser = Parser()' \
    '_HEADER_RE = re.compile(r"^(#{1,6})\s+(.+)$", re.MULTILINE)' \
    '' \
    'def extract_sections(md):' \
    '    if not md.strip(): return []' \
    '    lines = md.split("\n"); secs = []; cur_t = ""; cur_l = 0; cur_ln = []' \
    '    for ln in lines:' \
    '        m = _HEADER_RE.match(ln)' \
    '        if m:' \
    '            c = "\n".join(cur_ln).strip()' \
    '            if c: secs.append({"title":cur_t,"content":c,"level":cur_l})' \
    '            cur_t = m.group(2).strip(); cur_l = len(m.group(1)); cur_ln = []' \
    '        else: cur_ln.append(ln)' \
    '    c = "\n".join(cur_ln).strip()' \
    '    if c or not secs: secs.append({"title":cur_t,"content":c,"level":cur_l})' \
    '    return secs' \
    '' \
    'def parse_multipart(body, boundary):' \
    '    bb = boundary.encode("ascii"); fc = None; fn = "unknown"; ft = ""' \
    '    for part in body.split(b"--" + bb):' \
    '        if b"Content-Disposition" not in part: continue' \
    '        he = part.find(b"\r\n\r\n")' \
    '        if he == -1: continue' \
    '        hs = part[:he].decode("utf-8","ignore"); ct = part[he+4:].rstrip(b"\r\n--")' \
    '        if '\''name="file"'\'' in hs: fc = ct; m = re.search(r'\''filename="([^"]*)"'\'', hs);' \
    '            if m: fn = m.group(1)' \
    '        elif '\''name="file_type"'\'' in hs: ft = ct.decode("utf-8","ignore").strip()' \
    '    return fc, fn, ft' \
    '' \
    'class H(BaseHTTPRequestHandler):' \
    '    def do_GET(self):' \
    '        if self.path=="/health": self._j(200,{"status":"ok","service":"docreader-http-bridge"})' \
    '        else: self._j(404,{"error":"not found"})' \
    '    def do_POST(self):' \
    '        if self.path!="/parse": self._j(404,{"error":"not found"}); return' \
    '        ct = self.headers.get("Content-Type","")' \
    '        if "multipart/form-data" not in ct: self._j(400,{"error":"need multipart/form-data"}); return' \
    '        bm = re.search(r"boundary=([^;\s]+)", ct)' \
    '        if not bm: self._j(400,{"error":"missing boundary"}); return' \
    '        bd = bm.group(1).strip("\"")' \
    '        cl = int(self.headers.get("Content-Length",0))' \
    '        if cl==0: self._j(400,{"error":"empty body"}); return' \
    '        body = self.rfile.read(cl)' \
    '        fc,fn,ft = parse_multipart(body, bd)' \
    '        if fc is None or len(fc)==0: self._j(400,{"error":"no file"}); return' \
    '        if not ft and "." in fn: ft = fn.rsplit(".",1)[-1].lower()' \
    '        if not ft: ft = "txt"' \
    '        logger.info("parsing: %s (type=%s, size=%d)", fn, ft, len(fc))' \
    '        try:' \
    '            start = time.time()' \
    '            result = parser.parse_file(fn, ft, fc)' \
    '            elapsed = time.time()-start' \
    '            sections = extract_sections(result.content)' \
    '            meta = {str(k):str(v) for k,v in (result.metadata or {}).items()}' \
    '            resp = {"full_text":result.content,"sections":sections,"metadata":meta,"duration":round(elapsed,3)}' \
    '            logger.info("done: %s, %.2fs, %d sections", fn, elapsed, len(sections))' \
    '            self._j(200, resp)' \
    '        except Exception as e:' \
    '            logger.error("parse error: %s", e, exc_info=True)' \
    '            self._j(500, {"error": f"parse failed: {e}"})' \
    '    def _j(self, status, data):' \
    '        body = json.dumps(data, ensure_ascii=False).encode("utf-8")' \
    '        self.send_response(status)' \
    '        self.send_header("Content-Type","application/json; charset=utf-8")' \
    '        self.send_header("Content-Length", str(len(body)))' \
    '        self.end_headers(); self.wfile.write(body)' \
    '    def log_message(self, fmt, *args):' \
    '        logger.debug("%s - %s", self.client_address[0], fmt % args)' \
    '' \
    'if __name__=="__main__":' \
    '    port = int(os.environ.get("HTTP_PORT","50052"))' \
    '    srv = HTTPServer(("0.0.0.0",port), H)' \
    '    logger.info("HTTP bridge on port %d", port)' \
    '    try: srv.serve_forever()' \
    '    except KeyboardInterrupt: srv.shutdown()' \
    > /app/http_bridge.py

# 设置 Python 路径，确保 docreader 包可导入
ENV PYTHONPATH=/app

# 暴露 gRPC（50051）和 HTTP 桥接（50052）端口
EXPOSE 50051 50052

# 启动 gRPC 服务 + HTTP 桥接服务
CMD ["sh", "-c", "echo 'Starting gRPC server...' && python main.py & echo 'Starting HTTP bridge...' && python http_bridge.py & wait"]
