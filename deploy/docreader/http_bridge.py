#!/usr/bin/env python3
"""WeKnora docreader HTTP 桥接服务。

为 gRPC-only 的 docreader 提供 REST API：
  GET  /health  — 健康检查
  POST /parse   — 文档解析（multipart/form-data）
"""

import json
import logging
import os
import re
import sys
import time
from http.server import HTTPServer, BaseHTTPRequestHandler
from io import BytesIO

# 确保 docreader 包在 Python path 中
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from docreader.parser.parser import Parser

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [http_bridge] %(levelname)s %(message)s",
)
logger = logging.getLogger("http_bridge")

# 初始化解析器（全局单例，避免每次请求重新加载模型）
parser = Parser()

# Markdown 标题正则（用于提取章节结构）
_HEADER_RE = re.compile(r"^(#{1,6})\s+(.+)$", re.MULTILINE)


def extract_sections(markdown_text: str) -> list[dict]:
    """从 markdown 文本中提取章节结构。

    以 # 标题行作为章节边界，标题之前的非空内容归入
    前一个章节。无标题的纯文本作为单个无标题章节返回。
    """
    if not markdown_text.strip():
        return []

    lines = markdown_text.split("\n")
    sections: list[dict] = []
    current_title = ""
    current_level = 0
    current_lines: list[str] = []

    for line in lines:
        m = _HEADER_RE.match(line)
        if m:
            # 保存前一个章节
            content = "\n".join(current_lines).strip()
            if content:
                sections.append({
                    "title": current_title,
                    "content": content,
                    "level": current_level,
                })
            current_title = m.group(2).strip()
            current_level = len(m.group(1))
            current_lines = []
        else:
            current_lines.append(line)

    # 保存最后一个章节
    content = "\n".join(current_lines).strip()
    if content or not sections:
        sections.append({
            "title": current_title,
            "content": content,
            "level": current_level,
        })

    return sections


def parse_multipart(body: bytes, boundary: str) -> tuple[bytes | None, str, str]:
    """简易 multipart/form-data 解析器。

    Returns:
        (file_content, file_name, file_type_hint)
        若未找到 file 字段则返回 (None, "", "")
    """
    boundary_bytes = boundary.encode("ascii")
    file_content = None
    file_name = "unknown"
    file_type = ""

    # 按边界切分
    parts = body.split(b"--" + boundary_bytes)
    for part in parts:
        if b"Content-Disposition" not in part:
            continue

        # 分离头部和内容（使用 \r\n\r\n 分隔）
        header_end = part.find(b"\r\n\r\n")
        if header_end == -1:
            continue

        header_section = part[:header_end].decode("utf-8", errors="ignore")
        content = part[header_end + 4:]

        # 去除结尾的换行和边界符
        content = content.rstrip(b"\r\n--")

        if 'name="file"' in header_section:
            file_content = content
            # 提取原始文件名
            fn_match = re.search(r'filename="([^"]*)"', header_section)
            if fn_match:
                file_name = fn_match.group(1)
        elif 'name="file_type"' in header_section:
            file_type = content.decode("utf-8", errors="ignore").strip()

    return file_content, file_name, file_type


class ParseHandler(BaseHTTPRequestHandler):
    """HTTP 请求处理器"""

    def do_GET(self) -> None:
        if self.path == "/health":
            self._json_response(200, {"status": "ok", "service": "docreader-http-bridge"})
        else:
            self._json_response(404, {"error": "not found"})

    def do_POST(self) -> None:
        if self.path != "/parse":
            self._json_response(404, {"error": "not found"})
            return

        content_type = self.headers.get("Content-Type", "")
        if "multipart/form-data" not in content_type:
            self._json_response(400, {"error": "需要 multipart/form-data"})
            return

        # 提取 boundary
        boundary_match = re.search(r"boundary=([^;\s]+)", content_type)
        if not boundary_match:
            self._json_response(400, {"error": "缺少 boundary 参数"})
            return
        boundary = boundary_match.group(1).strip('"')

        content_length = int(self.headers.get("Content-Length", 0))
        if content_length == 0:
            self._json_response(400, {"error": "请求体为空"})
            return

        body = self.rfile.read(content_length)
        file_content, file_name, file_type = parse_multipart(body, boundary)

        if file_content is None or len(file_content) == 0:
            self._json_response(400, {"error": "未找到上传文件"})
            return

        # 从文件名推断类型
        if not file_type and "." in file_name:
            file_type = file_name.rsplit(".", 1)[-1].lower()
        if not file_type:
            file_type = "txt"

        logger.info("解析文件: %s (type=%s, size=%d bytes)", file_name, file_type, len(file_content))

        try:
            start = time.time()
            result = parser.parse_file(file_name, file_type, file_content)
            elapsed = time.time() - start

            sections = extract_sections(result.content)
            metadata = result.metadata or {}

            # 将 metadata 值统一转为字符串
            str_metadata = {str(k): str(v) for k, v in metadata.items()}

            response = {
                "full_text": result.content,
                "sections": sections,
                "metadata": str_metadata,
                "duration": round(elapsed, 3),
            }

            logger.info(
                "解析完成: %s, 耗时 %.2fs, %d 个章节",
                file_name, elapsed, len(sections),
            )
            self._json_response(200, response)

        except Exception as e:
            logger.error("解析失败: %s", e, exc_info=True)
            self._json_response(500, {"error": f"解析失败: {str(e)}"})

    def _json_response(self, status: int, data: dict) -> None:
        """发送 JSON 响应"""
        body = json.dumps(data, ensure_ascii=False).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    # 抑制默认的 stderr 日志，改用 logging
    def log_message(self, format: str, *args) -> None:  # noqa: A002
        logger.debug("%s - %s", self.client_address[0], format % args)


def main() -> None:
    port = int(os.environ.get("HTTP_PORT", "50052"))
    server = HTTPServer(("0.0.0.0", port), ParseHandler)
    logger.info("HTTP 桥接服务启动，监听端口 %d", port)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        logger.info("收到终止信号，关闭服务")
        server.shutdown()


if __name__ == "__main__":
    main()
