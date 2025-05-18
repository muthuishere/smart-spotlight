// SearchContainer.jsx
import { useState, useEffect, useCallback } from "react";
import {
  WindowHide,
  WindowSetSize,
  LogInfo,
  EventsOn,
   EventsOff
} from "../../../wailsjs/runtime/runtime";
import ConfirmationDialog from "./ConfirmationDialog";
import SearchInput      from "./SearchInput";
import MarkdownResponse from "./MarkdownResponse";
import { SearchWithMCP,ConfirmTool } from "../../../wailsjs/go/backend/App";
import { getWindowSize,resizeForError,resizeForResponse,resizeToDefault } from "../../config/windowConfig";

export default function SearchContainer() {
  const [searchQuery, setSearchQuery]   = useState("");
  const [response,    setResponse]      = useState(null);
  const [isLoading,   setIsLoading]     = useState(false);
  const [error,       setError]         = useState(null);
  const [confirm, setConfirm] = useState(null); // {token, message}
  const [showConfirm,setShowConfirm] = useState(false)


  async function onConfirmationRequiredEvent(data){

    setConfirm({
      token: data.token,
      markdown:
        `### Confirm operation\n\n` +
        `**Tool:** \`${data.tool}\`\n\n` +
        "\nProceed?",
    });
    setIsLoading(false);
    setShowConfirm(true);
    await resizeForResponse()
  }


  useEffect(()=>{
    onConfirmationRequiredEvent({token:"123",tool:"delete tool",args:[23,34]})
  },[])

  /* ------------------------------------------------------------------ */
  /* PromptEvent listener                                               */
  /* ------------------------------------------------------------------ */
  useEffect(() => {
    const onPromptEvent = async (ev) => {
      switch (ev.Type) {
        case "confirmation_required":
          await onConfirmationRequiredEvent(ev.Data)
          break;

        case "tool_use":
          // optional: progress indicator
          break;

        case "error":
          setError(ev.Data);
          setIsLoading(false);
          setResponse(null);
          {
            const { width, height } = await getWindowSize({ hasError: true });
            await WindowSetSize(width, height);
          }
          break;

        case "final_result":
          // ev.Data is HistoryMessage -> ev.Data.Content[0].Text (simplest)
     
          setResponse(ev.Data);        // already markdown
          setIsLoading(false);
          setError(null);

          {
            const { width, height } = await getWindowSize({ hasResponse: true });
            await WindowSetSize(width, height);
          }
          break;

        case "authorization_required":

          // Show modal / toast here
          break;
      }
    }

    EventsOn("PromptEvent", onPromptEvent);

    return () => {
      EventsOff("PromptEvent", onPromptEvent);
    };
  }, []);



 

  /* ------------------------------------------------------------------ */
  /* Fire prompt                                                        */
  /* ------------------------------------------------------------------ */
  const triggerPrompt = useCallback(async (query) => {
    if (!query.trim()) return;

    setIsLoading(true);
    setError(null);
    setResponse(null);

    try {
      await SearchWithMCP(query);   // returns quickly
      LogInfo("Prompt sent to backend");
    } catch (err) {
      setError(String(err));
      setIsLoading(false);
      resizeForError();
    }
  }, []);
  /* ------------------------------------------------------------------ */
  /* Handlers                                                           */
  /* ------------------------------------------------------------------ */
  const handleSearchChange = (e) => {
    setSearchQuery(e.target.value);
    setResponse(null);
    setError(null);
     resizeToDefault().then(()=>{});
  };

  const handleEscape = (e) => {
    if (e.key === "Escape") WindowHide();
    else if (e.key === "Enter") triggerPrompt(searchQuery);
  };


  const handleConfirmationChoice =async (ok)=>{
    setIsLoading(true)
   
    ConfirmTool(confirm.token, ok).then(()=>{})
    await resizeToDefault()
    await setShowConfirm(false)
    setConfirm(null);
    if (!ok) setIsLoading(false);
    

  }

  const handleCopySuccess = () => WindowHide();
  /* ------------------------------------------------------------------ */
  /* UI                                                                 */
  /* ------------------------------------------------------------------ */
  return (
    <div className="flex flex-col">
      <SearchInput
        value={searchQuery}
        onChange={handleSearchChange}
        onSearch={() => triggerPrompt(searchQuery)}
        placeholder="Ask anything…"
        isLoading={isLoading}
        onKeyDown={handleEscape}
        hasResponse={!!response}  
      />  

      {error && (
        <div className="px-4 py-2 text-destructive text-sm">{error}</div>
      )}

      {response && (
        <div className="overflow-y-auto flex-1" style={{ maxHeight: "calc(100vh - 120px)" }}>
          <MarkdownResponse content={response} onCopy={handleCopySuccess} />
        </div>
      )}

{showConfirm && (
        <div className="overflow-y-auto flex-1" style={{ maxHeight: "calc(100vh - 120px)" }}>
           <ConfirmationDialog
        content={confirm.markdown}
 
        onChoice={handleConfirmationChoice}
      />
        </div>
      )}




    </div>
  );
}
