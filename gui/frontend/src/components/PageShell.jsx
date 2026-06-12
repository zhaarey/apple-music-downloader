/**
 * Scrollable tab page shell — fills available main area and grows with window width.
 */
export default function PageShell({ children, wide = false, className = '', ...rest }) {
  const widthClass = wide
    ? 'max-w-content xl:max-w-[90rem] 2xl:max-w-[min(100rem,calc(100vw-4rem))]'
    : 'max-w-content xl:max-w-[80rem]'

  return (
    <div
      className={`min-h-0 flex-1 overflow-y-auto overflow-x-hidden overscroll-y-contain ${className}`}
      {...rest}
    >
      <div className={`mx-auto flex w-full flex-col gap-4 pb-8 ${widthClass}`}>{children}</div>
    </div>
  )
}
