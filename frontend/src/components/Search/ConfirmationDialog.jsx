import React from 'react';
import { marked } from 'marked';
import hljs from 'highlight.js';
import 'highlight.js/styles/github-dark.css';
import { AlertTriangle } from 'lucide-react';

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


export default function ConfirmationDialog({ content, onChoice }) {
  return (
    <div className="bg-secondary/20 rounded-lg shadow-lg mx-4 my-3 overflow-hidden">
      <div className="border-l-4 border-primary p-6">
        <div className="flex items-start space-x-4">
          <AlertTriangle className="h-6 w-6 text-primary flex-shrink-0" />
          <div className="flex-1">
            <div
              className="markdown-content prose prose-invert max-w-none prose-p:my-2 prose-headings:text-primary prose-headings:font-semibold prose-pre:bg-secondary/30 prose-pre:rounded-lg prose-code:text-foreground prose-code:bg-secondary/30 prose-code:rounded prose-code:px-1.5 prose-code:py-0.5"
              dangerouslySetInnerHTML={{ __html: marked(content) }}
            />
            
            <div className="mt-8 flex justify-end space-x-4">
              <button
                onClick={() => onChoice(false)}
                className="px-5 py-2 rounded-md bg-secondary hover:bg-secondary/80 transition-colors text-foreground border border-secondary/50"
              >
                Cancel
              </button>
              <button
                onClick={() => onChoice(true)}
                className="px-5 py-2 rounded-md bg-primary hover:bg-primary/80 transition-colors text-primary-foreground font-medium"
              >
                Proceed
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}