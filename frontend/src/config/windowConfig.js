import { LogInfo } from "../../wailsjs/runtime/runtime";

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