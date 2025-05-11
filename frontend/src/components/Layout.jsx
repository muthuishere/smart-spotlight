import React from 'react';
import { Outlet } from 'react-router-dom';

export default function Layout() {
  return (
    <div className="h-full w-full bg-background/80 backdrop-blur-xl">
      <Outlet />
    </div>
  );
}