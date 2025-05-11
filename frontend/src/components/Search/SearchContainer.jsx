import { useState } from "react";
import { LogInfo, WindowHide, WindowSetSize } from "../../../wailsjs/runtime/runtime";
import SearchInput from "./SearchInput";
import MarkdownResponse from "./MarkdownResponse";
import { SearchWithLLM } from "../../../wailsjs/go/backend/App";
import { getWindowSize } from "../../config/windowConfig";

function SearchContainer() {
  const [searchQuery, setSearchQuery] = useState("");
  const [response, setResponse] = useState(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState(null);
  const [suggestionsVisible, setSuggestionsVisible] = useState(false);

  const handleSearch = async (query) => {
    if (!query.trim()) return;
    
    setIsLoading(true);
    setError(null);
    try {
      const result = await SearchWithLLM(query);
      LogInfo("Search result:"+JSON.stringify(result));
      if (result.error) {
        setError(result.error);
        setResponse(null);
        const { width, height } = await getWindowSize({ hasError: true });
        await WindowSetSize(width, height);
      } else {
        LogInfo("Search result content:"+JSON.stringify(result.content));
        setResponse(result.content);
        setError(null);
        const { width, height } = await getWindowSize({ hasResponse: true });
        await WindowSetSize(width, height);
      }
    } catch (err) {
      setError(err.toString());
      setResponse(null);
      const { width, height } = await getWindowSize({ hasError: true });
      await WindowSetSize(width, height);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSearchChange = (e) => {
    setSearchQuery(e.target.value);
    // Clear response when typing
    setResponse(null);
    setError(null);
    // Reset window size to default
    getWindowSize({ hasError: false, hasResponse: false }).then(({ width, height }) => {
      WindowSetSize(width, height);
    });
  };

  const handleCopySuccess = () => {
    WindowHide();
  };

  const handleEscape = (e) => {
    if (e.key === "Escape") {
      WindowHide();
    } else if (e.key === "Enter" && searchQuery.trim()) {
      handleSearch(searchQuery);
    }
  };

  return (
    <div className="flex flex-col">
      <SearchInput
        value={searchQuery}
        onChange={handleSearchChange}
        onSearch={() => handleSearch(searchQuery)}
        placeholder="Ask anything..."
        isLoading={isLoading}
        onSuggestionsVisibilityChange={setSuggestionsVisible}
        hasResponse={!!response}
        onKeyDown={handleEscape}
      />
      {error && (
        <div className="px-4 py-2 text-destructive text-sm">
          {error}
        </div>
      )}
      {response && (
        <div className="overflow-y-auto flex-1" style={{ maxHeight: 'calc(100vh - 120px)' }}>
          <MarkdownResponse 
            content={response}
            onCopy={handleCopySuccess}
          />
        </div>
      )}
    </div>
  );
}

export default SearchContainer;