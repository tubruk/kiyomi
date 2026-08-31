import { FitMode } from '../../hooks/useReaderFitMode';

export function getFitModeClasses(fitMode: FitMode, baseClasses = ''): string {
  const fitWidthClass = 'max-w-full h-auto';
  const fitHeightClass = 'max-h-[calc(100vh-7.5rem)] w-auto max-w-full object-contain';
  const fitOriginalClass = 'max-w-none h-auto';

  let fitClass: string;
  switch (fitMode) {
    case 'fit-width':
      fitClass = fitWidthClass;
      break;
    case 'fit-original':
      fitClass = fitOriginalClass;
      break;
    case 'fit-height':
    default:
      fitClass = fitHeightClass;
      break;
  }

  // Replace fit-related classes in baseClasses, then append fitClass
  const withoutFit = baseClasses
    .replace(/max-w-full|max-w-none/g, '')
    .replace(/max-h-\[calc\(100vh-7\.5rem\)\]/g, '')
    .replace(/w-auto|h-auto/g, '')
    .replace(/object-contain/g, '')
    .replace(/\s+/g, ' ')
    .trim();

  return `${withoutFit} ${fitClass}`.trim();
}
