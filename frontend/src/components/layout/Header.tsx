export function Header({ title }: { title: string }) {
  return (
    <header className="flex h-14 items-center justify-between border-b border-gray-200 px-6">
      <h1 className="text-base font-semibold text-gray-900">{title}</h1>
      <div className="flex items-center gap-2 text-xs text-gray-500">
        <span className="inline-block h-2 w-2 rounded-full bg-green-500" />
        服务已连接
      </div>
    </header>
  );
}
