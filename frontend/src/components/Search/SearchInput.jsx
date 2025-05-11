import React, { useState, useEffect, useCallback, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { Settings } from 'lucide-react';
import { GetSearchHistory } from '../../../wailsjs/go/backend/App';
import { WindowSetSize, WindowHide } from '../../../wailsjs/runtime/runtime';
import { getWindowSize } from '../../config/windowConfig';

function SearchInput({ value, onChange, onSearch, placeholder = "Search...", isLoading, onSuggestionsVisibilityChange, hasResponse }) {
  const navigate = useNavigate();
  const [suggestions, setSuggestions] = useState([]);
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const inputRef = useRef(null);

  // Focus input on mount and when window gains focus
  useEffect(() => {
    inputRef.current?.focus();
    
    const handleFocus = () => {
      inputRef.current?.focus();
    };
    
    window.addEventListener('focus', handleFocus);
    return () => window.removeEventListener('focus', handleFocus);
  }, []);

  // Handle window size when suggestions visibility changes
  useEffect(() => {
    const updateWindowSize = async () => {
      const { width, height } = await getWindowSize({
        hasSuggestions: suggestions.length > 0,
        hasResponse: hasResponse
      });
      await WindowSetSize(width, height);
    };
    updateWindowSize();
    onSuggestionsVisibilityChange?.(suggestions.length > 0);
  }, [suggestions.length, hasResponse, onSuggestionsVisibilityChange]);

  // Clear suggestions when response is shown or loading starts
  useEffect(() => {
    if (isLoading || hasResponse) {
      setSuggestions([]);
      setSelectedIndex(-1);
      onSuggestionsVisibilityChange?.(false);
    }
  }, [isLoading, hasResponse, onSuggestionsVisibilityChange]);

  const debouncedFetchSuggestions = useCallback(
    (() => {
      let timeout;
      return (query) => {
        if (timeout) clearTimeout(timeout);
        timeout = setTimeout(async () => {
          // Only fetch suggestions if there's no response showing
          if (query.trim() && !isLoading && !hasResponse) {
            try {
              const history = await GetSearchHistory(query);
              setSuggestions(history || []);
            } catch {
              setSuggestions([]);
            }
          } else {
            setSuggestions([]);
          }
        }, 150);
      };
    })(),
    [isLoading, hasResponse]
  );

  useEffect(() => {
    debouncedFetchSuggestions(value);
  }, [value, debouncedFetchSuggestions]);

  const handleKeyDown = (e) => {
    if (e.key === 'Escape') {
      e.preventDefault();
      WindowHide();
      return;
    }
    
    if (e.key === 'Enter') {
      if (selectedIndex >= 0 && suggestions.length > 0) {
        e.preventDefault();
        const selected = suggestions[selectedIndex];
        onChange({ target: { value: selected.query } });
        onSearch(selected.query);
      } else if (value.trim()) {
        onSearch(value);
      }
      setSuggestions([]);
      setSelectedIndex(-1);
      return;
    }

    if (suggestions.length > 0) {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setSelectedIndex(prev => Math.min(prev + 1, suggestions.length - 1));
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        setSelectedIndex(prev => prev > -1 ? prev - 1 : suggestions.length - 1);
      }
    }
  };

  const handleInputChange = (e) => {
    onChange(e);
    setSelectedIndex(-1);
  };

  const handleSuggestionClick = (suggestion) => {
    onChange({ target: { value: suggestion.query } });
    onSearch(suggestion.query);
    setSuggestions([]);
    setSelectedIndex(-1);
  };

  return (
    <div className="relative flex items-center p-2">
      <div className="relative flex-1 mr-8">
        <input
          ref={inputRef}
          type="text"
          value={value}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          className="w-full px-3 py-1.5 bg-secondary/40 rounded-md border-none focus:outline-none focus:ring-1 focus:ring-secondary/60"
          autoComplete="off"
          autoCorrect="off"
          autoCapitalize="off"
          spellCheck="false"
          autoFocus
        />
        {isLoading && (
          <div className="absolute right-3 top-1/2 -translate-y-1/2">
            <div className="w-3 h-3 border-2 border-muted-foreground/30 border-t-muted-foreground/90 rounded-full animate-spin" />
          </div>
        )}
        {!isLoading && suggestions.length > 0 && (
          <div className="absolute w-full mt-1 py-1 bg-secondary/90 backdrop-blur-xl rounded-md shadow-lg max-h-[200px] overflow-y-auto">
            {suggestions.map((suggestion, index) => (
              <button
                key={suggestion.id}
                className={`w-full px-3 py-1.5 text-left hover:bg-secondary/60 ${
                  index === selectedIndex ? 'bg-secondary/60' : ''
                }`}
                onClick={() => handleSuggestionClick(suggestion)}
              >
                {suggestion.query}
              </button>
            ))}
          </div>
        )}
      </div>
      <button 
        onClick={() => navigate('/settings')}
        className="absolute right-2.5 p-1.5 rounded-md hover:bg-secondary/40 text-muted-foreground transition-colors"
        title="Settings"
      >
        <Settings className="w-3.5 h-3.5" />
      </button>
    </div>
  );
}

export default SearchInput;