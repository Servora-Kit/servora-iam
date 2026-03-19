export function AuthCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="min-h-screen flex items-center justify-center bg-[#eff1f5] dark:bg-[#1e1e2e]">
      <div className="w-full max-w-sm bg-white dark:bg-[#181825] rounded-xl shadow-sm border border-[#ccd0da] dark:border-[#313244] p-8">
        <div className="mb-6 flex items-center gap-2">
          <span className="text-lg font-semibold text-[#4c4f69] dark:text-[#cdd6f4]">Servora</span>
        </div>
        <h1 className="text-xl font-semibold text-[#4c4f69] dark:text-[#cdd6f4] mb-6">{title}</h1>
        {children}
      </div>
    </div>
  )
}
