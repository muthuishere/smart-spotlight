import React from 'react';

function SearchResultItem({ title, description, onClick, isSelected }) {
  return (
    <div 
      className={`h-[58px] px-4 flex flex-row items-center cursor-pointer rounded-lg transition-colors duration-200 gap-3
        ${isSelected ? 'bg-secondary hover:bg-secondary/90' : 'hover:bg-muted/50'}`}
      onClick={onClick}
      role="button"
      tabIndex={0}
    >
      <div className="w-10 h-10 flex items-center justify-center flex-shrink-0 rounded-full bg-primary/10">
        {/* Icon placeholder - first letter of title */}
        <span className="text-lg text-text/70">{title[0]}</span>
      </div>
      <div className="flex flex-col min-w-0 flex-grow text-left">
        <div className="text-base font-medium truncate text-text">
          {title}
        </div>
        {description && (
          <div className="text-sm truncate text-muted-foreground">
            {description}
          </div>
        )}
      </div>
    </div>
  );
}

function SearchResults({ results = [], onItemClick, selectedIndex = -1 }) {
  if (results.length === 0) return null;
  
  return (
    <div className="mt-2 max-h-[464px] overflow-y-auto scrollbar-thin scrollbar-thumb-muted scrollbar-track-transparent">
      {results.map((result, index) => (
        <SearchResultItem
          key={index}
          title={result.title}
          description={result.description}
          onClick={() => onItemClick(result)}
          isSelected={index === selectedIndex}
        />
      ))}
    </div>
  );
}

export default SearchResults;