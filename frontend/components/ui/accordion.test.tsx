import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";

import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";

describe("AccordionTrigger", () => {
  test("传入 onClick 时，点击既触发回调又展开内容", () => {
    const onClick = vi.fn();
    render(
      <Accordion>
        <AccordionItem value="item-1">
          <AccordionTrigger onClick={onClick}>标题</AccordionTrigger>
          <AccordionContent>内容</AccordionContent>
        </AccordionItem>
      </Accordion>,
    );

    // 初始收起：内容不可见
    expect(screen.queryByText("内容")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "标题" }));

    // 用户回调被调用，且内容展开
    expect(onClick).toHaveBeenCalledTimes(1);
    expect(screen.getByText("内容")).toBeInTheDocument();
  });

  test("再次点击能收起内容", () => {
    const onClick = vi.fn();
    render(
      <Accordion>
        <AccordionItem value="item-1">
          <AccordionTrigger onClick={onClick}>标题</AccordionTrigger>
          <AccordionContent>内容</AccordionContent>
        </AccordionItem>
      </Accordion>,
    );

    const trigger = screen.getByRole("button", { name: "标题" });

    fireEvent.click(trigger);
    expect(screen.getByText("内容")).toBeInTheDocument();

    fireEvent.click(trigger);
    expect(screen.queryByText("内容")).not.toBeInTheDocument();
    expect(onClick).toHaveBeenCalledTimes(2);
  });
});