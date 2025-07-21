import React from 'react';
import FilterForm from '../FilterForm';

/**
 * FiltersSection component for the agent form
 */
const FiltersSection = ({ formData, setFormData, metadata }) => {
  // Handle filter change
  const handleFilterChange = (index, updatedFilter) => {
    const updatedFilters = [...(formData.filters || [])];
    updatedFilters[index] = updatedFilter;
    setFormData({
      ...formData,
      filters: updatedFilters
    });
  };

  // Handle filter removal
  const handleFilterRemove = (index) => {
    const updatedFilters = [...(formData.filters || [])].filter((_, i) => i !== index);
    setFormData({
      ...formData,
      filters: updatedFilters
    });
  };

  // Handle adding an filter
  const handleAddFilter = () => {
    setFormData({
      ...formData,
      filters: [
        ...(formData.filters || []),
        { name: '', config: '{}' }
      ]
    });
  };

  return (
    <div className="filters-section">
      <h3 className='section-title'>Filters</h3>
      <p className="section-description">
        Jobs received by the agent must pass all filters and at least one trigger (if any are specified)
      </p>

      <FilterForm
        filters={formData.filters || []}
        onChange={handleFilterChange}
        onRemove={handleFilterRemove}
        onAdd={handleAddFilter}
        fieldGroups={metadata?.filters || []}
      />
      
      <div className="tutorial-link" style={{ marginTop: '1rem' }}>
        <p>
          <a 
            href="https://www.youtube.com/watch?v=d_we-AYksSw" 
            target="_blank" 
            rel="noopener noreferrer"
            style={{ color: '#007bff', textDecoration: 'none' }}
          >
            Watch this video to learn about filters & triggers
          </a>
        </p>
      </div>
      
    </div>
  );
};

export default FiltersSection;
