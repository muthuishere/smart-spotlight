
import {
  WindowHide,
  WindowSetSize,
  LogInfo,

} from "../../wailsjs/runtime";
// Hardcoded window configurations
const WINDOW_CONFIG = {
  default: {
    width: 600,
    minHeight: 60,
    maxHeight: 800
  },
  withError: {
    additionalHeight: 40
  },
  withSuggestions: {
    additionalHeight: 200
  },
  withResponse: {
    height: 600  // Fixed height for response
  }
};

function calculateWindowHeight({
  hasError,
  hasResponse,
  hasSuggestions
}) {
  if (hasResponse) {
    return WINDOW_CONFIG.withResponse.height;
  }

  let height = WINDOW_CONFIG.default.minHeight;

  if (hasError) {
    height += WINDOW_CONFIG.withError.additionalHeight;
  }

  if (hasSuggestions) {
    height = Math.min(height + WINDOW_CONFIG.withSuggestions.additionalHeight, WINDOW_CONFIG.default.maxHeight);
  }

  return Math.max(height, WINDOW_CONFIG.default.minHeight);
}

export async function getWindowSize({
  hasError = false,
  hasResponse = false,
  hasSuggestions = false
}) {
  const height = calculateWindowHeight({
    hasError,
    hasResponse,
    hasSuggestions
  });

  return {
    width: WINDOW_CONFIG.default.width,
    height
  };
}

export  function getDefaultWindowSize() {
  // 
  return {
    width: WINDOW_CONFIG.default.width,
    height:WINDOW_CONFIG.default.minHeight
  };
}

export const resizeForError = async () => {
    const { width, height } = await getWindowSize({ hasError: true });
    await WindowSetSize(width, height);
  };

  export const resizeForResponse = async () => {
    const { width, height } = await getWindowSize({ hasResponse: true });
    await WindowSetSize(width, height);
  };


  export const resizeForSuggestions = async () => {
    const { width, height } = await getWindowSize({ hasSuggestions: true });
    await WindowSetSize(width, height);
  };

  export const resizeToDefault = async () => {
    const { width, height } =  getDefaultWindowSize();
    await WindowSetSize(width, height);
  };