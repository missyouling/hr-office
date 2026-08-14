import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { fetchUserAvatar, resetUserAvatar, uploadUserAvatar } from "./avatar-api";

describe("头像 API 封装", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    localStorage.clear();
    fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  describe("fetchUserAvatar GET 加载", () => {
    it("携带 Bearer token 请求 /user/avatar 并返回 Blob", async () => {
      localStorage.setItem("token", "test-token");
      const blob = new Blob(["svg-content"], { type: "image/svg+xml" });
      // jsdom 的 Response.blob() 未实现，这里 mock 最小响应对象
      fetchMock.mockResolvedValue({
        ok: true,
        status: 200,
        blob: vi.fn().mockResolvedValue(blob),
      });

      const result = await fetchUserAvatar();

      expect(result).toBe(blob);
      expect(fetchMock).toHaveBeenCalledTimes(1);
      const [url, init] = fetchMock.mock.calls[0];
      expect(String(url)).toContain("/api/user/avatar");
      expect(init.headers).toEqual(expect.objectContaining({ Authorization: "Bearer test-token" }));
      // token 绝不进入 URL
      expect(String(url)).not.toContain("test-token");
    });

    it("非 2xx 时抛出后端错误文案", async () => {
      localStorage.setItem("token", "test-token");
      fetchMock.mockResolvedValue({
        ok: false,
        status: 500,
        json: vi.fn().mockResolvedValue({ error: "avatar storage unavailable" }),
      });

      await expect(fetchUserAvatar()).rejects.toThrow("avatar storage unavailable");
    });

    it("响应非 JSON 时回退默认文案", async () => {
      localStorage.setItem("token", "test-token");
      fetchMock.mockResolvedValue({
        ok: false,
        status: 502,
        json: vi.fn().mockRejectedValue(new Error("not json")),
      });

      await expect(fetchUserAvatar()).rejects.toThrow("头像加载失败");
    });
  });

  describe("uploadUserAvatar POST 上传", () => {
    it("以 multipart 字段 file 上传并返回元数据", async () => {
      localStorage.setItem("token", "test-token");
      const metadata = {
        type: "custom",
        seed: "seed-1",
        custom_file_id: 7,
        custom_content_type: "image/webp",
        content_type: "image/webp",
      };
      fetchMock.mockResolvedValue({
        ok: true,
        status: 200,
        json: vi.fn().mockResolvedValue(metadata),
      });

      const file = new File(["webp-bytes"], "avatar.webp", { type: "image/webp" });
      const result = await uploadUserAvatar(file);

      expect(result).toEqual(metadata);
      const [url, init] = fetchMock.mock.calls[0];
      expect(String(url)).toContain("/api/user/avatar");
      expect(init.method).toBe("POST");
      expect(init.body).toBeInstanceOf(FormData);
      const form = init.body as FormData;
      expect(form.get("file")).toBe(file);
      expect(init.headers).toEqual(expect.objectContaining({ Authorization: "Bearer test-token" }));
    });
  });

  describe("resetUserAvatar DELETE 恢复默认", () => {
    it("调用 DELETE /user/avatar 并返回元数据", async () => {
      localStorage.setItem("token", "test-token");
      const metadata = { type: "default", seed: "seed-1", content_type: "image/svg+xml" };
      fetchMock.mockResolvedValue({
        ok: true,
        status: 200,
        json: vi.fn().mockResolvedValue(metadata),
      });

      const result = await resetUserAvatar();

      expect(result).toEqual(metadata);
      const [url, init] = fetchMock.mock.calls[0];
      expect(String(url)).toContain("/api/user/avatar");
      expect(init.method).toBe("DELETE");
      expect(init.headers).toEqual(expect.objectContaining({ Authorization: "Bearer test-token" }));
    });
  });
});