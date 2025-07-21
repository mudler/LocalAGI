import React, { useState, useEffect } from 'react';
import { format } from 'date-fns';
import Header from "../components/Header";
import { usageApi } from "../utils/api";

const Usage = () => {
  const [usage, setUsage] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    document.title = "Usage - LocalAGI";
    return () => {
      document.title = "LocalAGI";
    };
  }, []);

  useEffect(() => {
    const fetchUsage = async () => {
      try {
        const data = await usageApi.getUsage();
        setUsage(data);
      } catch (error) {
        console.error('Error fetching usage:', error);
        setError('Failed to load usage data');
      } finally {
        setLoading(false);
      }
    };

    fetchUsage();
  }, []);

  const columns = [
    {
      title: 'Date',
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (date) => format(new Date(date), 'MMM d, h:mm a'),
      sorter: (a, b) => new Date(a.createdAt) - new Date(b.createdAt),
    },
    {
      title: 'Model',
      dataIndex: 'model',
      key: 'model',
      sorter: (a, b) => a.model.localeCompare(b.model),
    },
    {
      title: 'Total Tokens',
      dataIndex: 'totalTokens',
      key: 'totalTokens',
      sorter: (a, b) => a.totalTokens - b.totalTokens,
    },
    {
      title: 'Cost ($)',
      dataIndex: 'cost',
      key: 'cost',
      render: (cost) => {
        const num = Number(cost);
        return num === 0 ? '0' : num.toFixed(4);
      },
      sorter: (a, b) => a.cost - b.cost,
    },
  ];

  if (loading) {
    return <div className="loading-container">
              <div className="spinner"></div>
            </div>;
  }

  if (error) {
    return <div className="error">{error}</div>;
  }

  return (
    <div className="dashboard-container">
      <div className="main-content-area">
        <div className="header-container">
          <Header
            title="Usage"
            description="View detailed usage statistics and costs for your LLM interactions."
          />
        </div>

        <div className="section-box">
          <div className="table-container">
            {usage.length === 0 ? (
              <div className="no-usage-message">No usage yet</div>
            ) : (
              <table className="usage-table">
                <thead>
                  <tr>
                    {columns.map((column) => (
                      <th key={column.key}>{column.title}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {usage.map((record) => (
                    <tr key={record.id}>
                      {columns.map((column) => (
                        <td key={`${record.id}-${column.key}`}>
                          {column.render
                            ? column.render(record[column.dataIndex])
                            : record[column.dataIndex]}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default Usage; 