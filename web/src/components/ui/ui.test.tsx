// @vitest-environment jsdom
import { act, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import {
  Accordion,
  AccordionItem,
  Avatar,
  Button,
  DropdownMenu,
  DropdownMenuItem,
  EmptyState,
  ErrorState,
  FormField,
  IconButton,
  Input,
  Modal,
  Select,
  Switch,
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
  ToastProvider,
  Tooltip,
  useToast,
} from "./index";

describe("Button", () => {
  it("blocks clicks and reports busy while loading", async () => {
    const onClick = vi.fn();
    const { rerender } = render(<Button onClick={onClick}>Save</Button>);
    await userEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(onClick).toHaveBeenCalledTimes(1);

    rerender(
      <Button onClick={onClick} loading>
        Save
      </Button>,
    );
    const button = screen.getByRole("button", { name: "Save" });
    expect(button.getAttribute("aria-busy")).toBe("true");
    fireEvent.click(button);
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("defaults to type=button so it never submits a form by accident", () => {
    render(<Button>Go</Button>);
    expect(screen.getByRole("button").getAttribute("type")).toBe("button");
  });
});

describe("FormField", () => {
  it("links label, hint and required state to the control", () => {
    render(
      <FormField label="Email" hint="We never share it." required>
        <Input />
      </FormField>,
    );
    const input = screen.getByLabelText(/Email/);
    expect(input.getAttribute("aria-describedby")).toBe(screen.getByText("We never share it.").id);
    expect(input.hasAttribute("required")).toBe(true);
    expect(input.hasAttribute("aria-invalid")).toBe(false);
  });

  it("marks the control invalid and describes it with the error", () => {
    render(
      <FormField label="Target" error="Choose a target score.">
        <Select defaultValue="">
          <option value="">—</option>
        </Select>
      </FormField>,
    );
    const select = screen.getByLabelText("Target");
    expect(select.getAttribute("aria-invalid")).toBe("true");
    expect(select.getAttribute("aria-describedby")).toBe(
      screen.getByText("Choose a target score.").id,
    );
  });
});

describe("Switch", () => {
  it("toggles aria-checked and reports the new value", async () => {
    const onChange = vi.fn();
    render(<Switch label="Show timer" onCheckedChange={onChange} />);
    const control = screen.getByRole("switch", { name: "Show timer" });
    expect(control.getAttribute("aria-checked")).toBe("false");
    await userEvent.click(control);
    expect(control.getAttribute("aria-checked")).toBe("true");
    expect(onChange).toHaveBeenCalledWith(true);
  });
});

describe("Tabs", () => {
  it("switches panels with arrow keys, Home and End, skipping disabled tabs", async () => {
    render(
      <Tabs defaultValue="a">
        <TabsList aria-label="Views">
          <TabsTrigger value="a">A</TabsTrigger>
          <TabsTrigger value="b">B</TabsTrigger>
          <TabsTrigger value="x" disabled>
            X
          </TabsTrigger>
          <TabsTrigger value="c">C</TabsTrigger>
        </TabsList>
        <TabsPanel value="a">Panel A</TabsPanel>
        <TabsPanel value="b">Panel B</TabsPanel>
        <TabsPanel value="c">Panel C</TabsPanel>
      </Tabs>,
    );
    expect(screen.getByRole("tabpanel").textContent).toBe("Panel A");
    await userEvent.tab();
    expect(document.activeElement).toBe(screen.getByRole("tab", { name: "A" }));

    await userEvent.keyboard("{ArrowRight}");
    expect(screen.getByRole("tab", { name: "B" }).getAttribute("aria-selected")).toBe("true");
    expect(screen.getByRole("tabpanel").textContent).toBe("Panel B");

    await userEvent.keyboard("{ArrowRight}");
    expect(screen.getByRole("tabpanel").textContent).toBe("Panel C");
    await userEvent.keyboard("{Home}");
    expect(screen.getByRole("tabpanel").textContent).toBe("Panel A");
    await userEvent.keyboard("{ArrowLeft}");
    expect(screen.getByRole("tabpanel").textContent).toBe("Panel C");
  });
});

describe("Accordion", () => {
  it("opens one item at a time in single mode", async () => {
    render(
      <Accordion>
        <AccordionItem value="one" title="One">
          First
        </AccordionItem>
        <AccordionItem value="two" title="Two">
          Second
        </AccordionItem>
      </Accordion>,
    );
    const one = screen.getByRole("button", { name: "One" });
    const two = screen.getByRole("button", { name: "Two" });
    await userEvent.click(one);
    expect(one.getAttribute("aria-expanded")).toBe("true");
    expect(screen.getByText("First").hidden).toBe(false);
    await userEvent.click(two);
    expect(one.getAttribute("aria-expanded")).toBe("false");
    expect(two.getAttribute("aria-expanded")).toBe("true");
  });
});

describe("Modal", () => {
  function Harness({ dismissible = true }: { dismissible?: boolean }) {
    const [open, setOpen] = useState(false);
    return (
      <>
        <Button onClick={() => setOpen(true)}>Open</Button>
        <Modal open={open} onClose={() => setOpen(false)} title="Confirm" dismissible={dismissible}>
          Body
        </Modal>
      </>
    );
  }

  it("opens as a labelled dialog and closes with the close button and Escape", async () => {
    render(<Harness />);
    await userEvent.click(screen.getByRole("button", { name: "Open" }));
    const dialog = screen.getByRole("dialog", { name: "Confirm" });
    expect(dialog.hasAttribute("open")).toBe(true);
    expect(document.body.style.overflow).toBe("hidden");

    await userEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(dialog.hasAttribute("open")).toBe(false);
    expect(document.body.style.overflow).toBe("");

    await userEvent.click(screen.getByRole("button", { name: "Open" }));
    fireEvent(dialog, new Event("cancel", { cancelable: true }));
    expect(dialog.hasAttribute("open")).toBe(false);
  });

  it("ignores Escape when not dismissible", async () => {
    render(<Harness dismissible={false} />);
    await userEvent.click(screen.getByRole("button", { name: "Open" }));
    const dialog = screen.getByRole("dialog", { name: "Confirm" });
    fireEvent(dialog, new Event("cancel", { cancelable: true }));
    expect(dialog.hasAttribute("open")).toBe(true);
    expect(screen.queryByRole("button", { name: "Close" })).toBeNull();
  });
});

describe("DropdownMenu", () => {
  function Menu({ onEdit, onDelete }: { onEdit: () => void; onDelete: () => void }) {
    return (
      <DropdownMenu
        trigger={(props) => (
          <IconButton {...props} aria-label="Actions">
            …
          </IconButton>
        )}
      >
        <DropdownMenuItem onSelect={onEdit}>Edit</DropdownMenuItem>
        <DropdownMenuItem disabled>Move</DropdownMenuItem>
        <DropdownMenuItem onSelect={onDelete}>Delete</DropdownMenuItem>
      </DropdownMenu>
    );
  }

  it("supports the menu button keyboard pattern", async () => {
    const onEdit = vi.fn();
    const onDelete = vi.fn();
    render(<Menu onEdit={onEdit} onDelete={onDelete} />);
    const trigger = screen.getByRole("button", { name: "Actions" });
    expect(trigger.getAttribute("aria-expanded")).toBe("false");

    trigger.focus();
    await userEvent.keyboard("{ArrowDown}");
    expect(trigger.getAttribute("aria-expanded")).toBe("true");
    expect(document.activeElement?.textContent).toBe("Edit");

    await userEvent.keyboard("{ArrowDown}");
    expect(document.activeElement?.textContent).toBe("Delete"); // disabled item skipped
    await userEvent.keyboard("{Enter}");
    expect(onDelete).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("menu")).toBeNull();
    expect(document.activeElement).toBe(trigger);

    await userEvent.keyboard("{ArrowDown}");
    await userEvent.keyboard("{Escape}");
    expect(screen.queryByRole("menu")).toBeNull();
    expect(document.activeElement).toBe(trigger);
    expect(onEdit).not.toHaveBeenCalled();
  });

  it("closes on an outside click", async () => {
    render(
      <>
        <Menu onEdit={() => {}} onDelete={() => {}} />
        <p>Outside</p>
      </>,
    );
    await userEvent.click(screen.getByRole("button", { name: "Actions" }));
    expect(screen.getByRole("menu")).toBeTruthy();
    await userEvent.click(screen.getByText("Outside"));
    expect(screen.queryByRole("menu")).toBeNull();
  });
});

describe("Tooltip", () => {
  it("shows on keyboard focus, describes the trigger and hides on Escape", async () => {
    render(
      <Tooltip content="Search questions">
        <IconButton aria-label="Search">S</IconButton>
      </Tooltip>,
    );
    await userEvent.tab();
    const tooltip = await screen.findByRole("tooltip");
    expect(tooltip.textContent).toBe("Search questions");
    expect(screen.getByRole("button").getAttribute("aria-describedby")).toBe(tooltip.id);
    await userEvent.keyboard("{Escape}");
    expect(screen.queryByRole("tooltip")).toBeNull();
  });
});

describe("Toast", () => {
  function Trigger() {
    const { toast } = useToast();
    return (
      <>
        <Button onClick={() => toast({ title: "Saved", variant: "success", duration: 1000 })}>
          Ok
        </Button>
        <Button onClick={() => toast({ title: "Failed", variant: "error", duration: 0 })}>
          Fail
        </Button>
      </>
    );
  }

  it("announces, auto-dismisses and can be closed", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      render(
        <ToastProvider>
          <Trigger />
        </ToastProvider>,
      );
      fireEvent.click(screen.getByRole("button", { name: "Ok" }));
      fireEvent.click(screen.getByRole("button", { name: "Fail" }));
      expect(screen.getByRole("status").textContent).toContain("Saved");
      expect(screen.getByRole("alert").textContent).toContain("Failed");

      act(() => {
        vi.advanceTimersByTime(1100);
      });
      expect(screen.queryByText("Saved")).toBeNull();
      expect(screen.getByText("Failed")).toBeTruthy(); // duration 0 stays

      fireEvent.click(screen.getByRole("button", { name: "Dismiss notification" }));
      expect(screen.queryByText("Failed")).toBeNull();
    } finally {
      vi.useRealTimers();
    }
  });
});

describe("Avatar", () => {
  it("falls back to initials without an image and after a load error", () => {
    const { rerender } = render(<Avatar name="Aziz Karimov" />);
    expect(screen.getByRole("img", { name: "Aziz Karimov" }).textContent).toBe("AK");

    rerender(<Avatar name="Madina" src="/broken.png" />);
    fireEvent.error(screen.getByRole("img", { name: "Madina" }));
    expect(screen.getByRole("img", { name: "Madina" }).textContent).toBe("M");
  });
});

describe("States", () => {
  it("renders an empty state with its next step", () => {
    render(<EmptyState title="No mistakes yet" action={<Button>Practice</Button>} />);
    expect(screen.getByText("No mistakes yet")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Practice" })).toBeTruthy();
  });

  it("renders an alert with retry and a reference, never raw errors", async () => {
    const onRetry = vi.fn();
    render(<ErrorState onRetry={onRetry} reference="req_1" />);
    expect(screen.getByRole("alert").textContent).toContain("Reference: req_1");
    await userEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });
});
