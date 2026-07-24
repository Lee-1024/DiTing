import { RouterProvider } from 'react-router-dom';
import { router } from '../router';

// App 渲染 App 组件。
export default function App() {
  return <RouterProvider router={router} />;
}
