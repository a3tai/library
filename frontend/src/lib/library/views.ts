import type {LibraryView} from '../../types';

export function viewLabelFor(view: LibraryView): string {
  switch (view) {
    case 'library':
      return 'Library';
    case 'recent':
      return 'Recently added';
    case 'categories':
      return 'Categories';
    case 'byauthor':
      return 'By author';
    case 'bysubject':
      return 'By subject';
    case 'unprocessed':
      return 'Unprocessed';
    case 'settings':
      return 'Settings';
  }
}
