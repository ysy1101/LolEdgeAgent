import { useParams } from 'react-router';

export default function BriefingDetail() {
  const { id } = useParams();
  return (
    <div>
      <h1 className="mb-6 text-2xl font-semibold text-gray-900">简报详情 #{id}</h1>
      <p className="text-gray-500">功能开发中。</p>
    </div>
  );
}
