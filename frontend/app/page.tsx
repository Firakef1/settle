import { Landing } from '../features/landing/Landing';
import { RedirectIfSignedIn } from '../features/landing/RedirectIfSignedIn';

// Public landing page. Signed-in users are sent on to /dashboard.
export default function HomePage() {
  return (
    <>
      <RedirectIfSignedIn />
      <Landing />
    </>
  );
}
