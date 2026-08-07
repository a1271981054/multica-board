import Mention, { type MentionOptions } from "@tiptap/extension-mention";
import { mergeAttributes } from "@tiptap/core";
import { ReactNodeViewRenderer } from "@tiptap/react";
import { SlashCommandView } from "./slash-command-view";
import { formatSlashCommandLabel } from "./slash-command-utils";
import { escapeMarkdownLabel } from "../utils/escape-markdown-label";

export const SlashCommandExtension = Mention.extend({
  name: "slashCommand",

  addOptions() {
    const parent = this.parent?.() ?? ({} as MentionOptions);
    return {
      HTMLAttributes: parent.HTMLAttributes ?? {},
      // Deleting a slash tag must remove the whole node without re-inserting
      // Mention's default trigger character ("@").
      deleteTriggerWithBackspace: true,
      renderText: ({ node }: { node: { attrs: { label?: string } } }) =>
        `/${formatSlashCommandLabel(node.attrs.label ?? "")}`,
      renderHTML: parent.renderHTML,
      suggestions: parent.suggestions ?? [],
      suggestion: { ...(parent.suggestion ?? {}), char: "/" },
    };
  },

  addNodeView() {
    return ReactNodeViewRenderer(SlashCommandView);
  },

  renderHTML({ node, HTMLAttributes }) {
    return [
      "span",
      mergeAttributes(
        { "data-type": "slash-command" },
        this.options.HTMLAttributes,
        HTMLAttributes,
      ),
      `/${formatSlashCommandLabel(node.attrs.label)}`,
    ];
  },

  addAttributes() {
    return { ...this.parent?.() };
  },

  parseHTML() {
    return [{ tag: 'span[data-type="slash-command"]' }];
  },

  markdownTokenizer: {
    name: "slashCommand",
    level: "inline" as const,
    start(src: string) {
      // Backslash is excluded from the char class so "\x" runs can only be
      // consumed by \\. — overlapping alternatives backtrack in 2^n ways
      // (ReDoS, GitHub #4881).
      return src.search(/\[\/(?:\\.|[^\]\\])+\]\(slash:\/\/skill\//);
    },
    tokenize(src: string) {
      const match = src.match(
        /^\[\/((?:\\.|[^\]\\])+)\]\(slash:\/\/skill\/([^)]+)\)/,
      );
      if (!match) return undefined;
      // Reverse escapeMarkdownLabel: unescape \[ \] \\ \( \). Must mirror the
      // escaped set exactly, or a label containing "\" fails to round-trip
      // through the linear tokenizer.
      const rawLabel = match[1]?.replace(/\\([[\]\\()])/g, "$1");
      return {
        type: "slashCommand",
        raw: match[0],
        attributes: { label: rawLabel, id: match[2] },
      };
    },
  },

  parseMarkdown: (token: any, helpers: any) => {
    return helpers.createNode("slashCommand", token.attributes);
  },

  renderMarkdown: (node: any) => {
    const { id, label } = node.attrs || {};
    // Escape [ ] \ ( ) so the label survives the linear tokenizer; must stay in
    // sync with the unescape in tokenize() above.
    const safeLabel = escapeMarkdownLabel(formatSlashCommandLabel(label));
    return `[/${safeLabel}](slash://skill/${id})`;
  },
});
