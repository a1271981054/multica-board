import { describe, expect, it } from "vitest";
import { summarizeTitle } from "./title-summary";

describe("summarizeTitle", () => {
  it("keeps a short description verbatim", () => {
    expect(summarizeTitle("修复登录页")).toBe("修复登录页");
  });

  it("strips markdown and collapses whitespace", () => {
    expect(summarizeTitle("**修复** [登录页](https://x.test) 的按钮")).toBe(
      "修复 登录页 的按钮",
    );
  });

  it("keeps whole sentences up to the 10-20 character window", () => {
    const title = summarizeTitle(
      "用户点击提交后页面一直转圈。登录接口返回 500，需要先排查服务端日志。",
    );
    expect(Array.from(title).length).toBeGreaterThanOrEqual(10);
    expect(Array.from(title).length).toBeLessThanOrEqual(20);
  });

  it("truncates a long single sentence to 20 characters", () => {
    const title = summarizeTitle(
      "这是一个非常非常长的没有标点符号的问题描述需要被压缩成看板标题",
    );
    expect(Array.from(title)).toHaveLength(20);
  });

  it("falls back to a Chinese placeholder for empty content", () => {
    expect(summarizeTitle("   ")).toBe("未命名任务");
  });
});
