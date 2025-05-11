import React, { useEffect, useCallback } from 'react';
import { marked } from 'marked';
import hljs from 'highlight.js';
import { Copy, Download } from 'lucide-react';
import 'highlight.js/styles/github-dark.css';
import { LogInfo } from '/wailsjs/runtime/runtime';

marked.setOptions({
  highlight: function(code, lang) {
    if (lang && hljs.getLanguage(lang)) {
      try {
        return hljs.highlight(code, { language: lang }).value;
      } catch (err) {
        return code;
      }
    }
    return code;
  },
  breaks: true,
  gfm: true
});

export default function MarkdownResponse({ content, onCopy }) {
  const handleCopy = useCallback(() => {
    navigator.clipboard.writeText(content);
    onCopy?.();
  }, [content, onCopy]);

  const handleCodeCopy = useCallback((code) => {
    navigator.clipboard.writeText(code);
  }, []);

  const handleDownload = useCallback(() => {
    const blob = new Blob([content], { type: 'text/markdown' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'response.md';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }, [content]);

  useEffect(() => {
    if (!content) return;

    LogInfo('Markdown content:'+ content);
    
    // Add copy buttons to code blocks after content is rendered
    const codeBlocks = document.querySelectorAll('.markdown-content pre code');
    codeBlocks.forEach(block => {
      const wrapper = document.createElement('div');
      wrapper.className = 'relative group';
      block.parentNode.insertBefore(wrapper, block);
      wrapper.appendChild(block);

      const button = document.createElement('button');
      button.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>';
      button.className = 'absolute right-2 top-2 p-1 rounded-md bg-secondary/20 text-secondary-foreground opacity-0 group-hover:opacity-100 transition-colors hover:bg-secondary/30';
      button.onclick = () => handleCodeCopy(block.textContent);
      wrapper.appendChild(button);
    });
  }, [content, handleCodeCopy]);

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-end gap-2 px-4 pt-2">
        <button
          onClick={handleCopy}
          className="p-1.5 rounded-md bg-secondary/20 text-secondary-foreground hover:bg-secondary/30 transition-colors"
          title="Copy full response"
        >
          <Copy className="w-3.5 h-3.5" />
        </button>
        <button
          onClick={handleDownload}
          className="p-1.5 rounded-md bg-secondary/20 text-secondary-foreground hover:bg-secondary/30 transition-colors"
          title="Download as markdown"
        >
          <Download className="w-3.5 h-3.5" />
        </button>
      </div>
      <div className="px-4 pb-4">
        <div
          className="markdown-content prose prose-invert max-w-none prose-p:my-2 prose-pre:my-2 prose-headings:my-2 prose-ul:my-2 prose-ol:my-2 prose-pre:bg-secondary/20 prose-pre:rounded-lg prose-code:text-foreground prose-code:bg-secondary/20 prose-code:rounded prose-code:px-1.5 prose-code:py-0.5"
          dangerouslySetInnerHTML={{ __html: marked(content) }}
        />
      </div>
    </div>
  );
}