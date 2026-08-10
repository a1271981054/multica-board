import { Editor } from "@tiptap/core";
import Document from "@tiptap/extension-document";
import Paragraph from "@tiptap/extension-paragraph";
import Text from "@tiptap/extension-text";
import { describe, expect, it } from "vitest";
import { SlashCommandExtension } from "./slash-command-extension";

function makeEditor() {
  return new Editor({
    extensions: [Document, Paragraph, Text, SlashCommandExtension],
    content: "",
  });
}

function fireBackspace(editor: Editor) {
  const view = editor.view;
  const event = new KeyboardEvent("keydown", {
    key: "Backspace",
    bubbles: true,
  });
  view.someProp("handleKeyDown", (handler) => handler(view, event));
}

describe("slash command deletion", () => {
  it("removes the tag without leaving a stray @ character", () => {
    const editor = makeEditor();
    editor.commands.insertContent([
      {
        type: "slashCommand",
        attrs: { label: "目标模式", id: "mode:目标模式" },
      },
    ]);
    editor.commands.focus("end");

    fireBackspace(editor);

    expect(editor.getText()).toBe("");
  });
});
